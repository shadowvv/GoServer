package enum

// 所有属性
const (
	AttributeBasicHp                                  int32 = iota + 1 //生命值
	AttributeBasicBaseHp                                               //基础生命值
	AttributeBasicBaseHpPercent                                        //基础生命值百分比
	AttributeBasicPhysicalAttack                                       //物理攻击
	AttributeBasicBasePhysicalAttack                                   //基础物理攻击
	AttributeBasicBasePhysicalAttackPercent                            //基础物理攻击百分比
	AttributeBasicMagicalAttack                                        //魔法攻击
	AttributeBasicBaseMagicalAttack                                    //基础魔法攻击
	AttributeBasicBaseMagicalAttackPercent                             //基础魔法攻击百分比
	AttributeBasicPhysicalDefense                                      //物理防御
	AttributeBasicBasePhysicalDefense                                  //基础物理防御
	AttributeBasicBasePhysicalDefensePercent                           //基础物理防御百分比
	AttributeBasicMagicalDefense                                       //魔法防御
	AttributeBasicBaseMagicalDefense                                   //基础魔法防御
	AttributeBasicBaseMagicalDefensePercent                            //基础魔法防御百分比
	AttributeBasicStrength                                             //力量
	AttributeBasicIntelligence                                         //智力
	AttributeBasicToughness                                            //韧性
	AttributeBasicToughnessEnhancement                                 //韧性提升
	AttributeBasicToughnessResistance                                  //韧性抗性
	AttributeBasicAttackSpeed                                          //攻击速度
	AttributeBasicMoveSpeed                                            //移动速度
	AttributeBasicPhysicalCritRate                                     //物理暴击率
	AttributeBasicMagicalCritRate                                      //魔法暴击率
	AttributeBasicPhysicalCritDamage                                   //物理暴击伤害
	AttributeBasicMagicalCritDamage                                    //魔法暴击伤害
	AttributeBasicPhysicalCritResistance                               //物理暴击抗性
	AttributeBasicMagicalCritResistance                                //魔法暴击抗性
	AttributeBasicPhysicalCritDamageResistance                         //物理暴击伤害抗性
	AttributeBasicMagicalCritDamageResistance                          //魔法暴击伤害抗性
	AttributeBasicPhysicalHit                                          //物理命中
	AttributeBasicMagicalHit                                           //魔法命中
	AttributeBasicPhysicalDodge                                        //物理闪避
	AttributeBasicMagicalDodge                                         //魔法闪避
	AttributeBasicPhysicalPenetration                                  //物理穿透
	AttributeBasicMagicalPenetration                                   //魔法穿透
	AttributeBasicPhysicalBlock                                        //物理格挡
	AttributeBasicMagicalBlock                                         //魔法格挡
	AttributeBasicPhysicalDamageIncrease                               //物理伤害增加
	AttributeBasicMagicalDamageIncrease                                //魔法伤害增加
	AttributeBasicPhysicalDamageReduction                              //物理伤害减少
	AttributeBasicMagicalDamageReduction                               //魔法伤害减少
	AttributeBasicFinalDamageIncrease                                  //最终伤害增加
	AttributeBasicFinalDamageReduction                                 //最终伤害减少
	AttributeBasicPvpDamageIncrease                                    //PVP伤害增加
	AttributeBasicPvpDamageReduction                                   //PVP伤害减少
	AttributeBasicSkillAttackPower                                     //技能攻击力
	AttributeBasicCombatPower                                          //战斗力
	AttributeBasicHpPercent                                            //生命百分比
	AttributeBasicPhysicalAttackPercent                                //物理攻击百分比
	AttributeBasicMagicalAttackPercent                                 //魔法攻击百分比
	AttributeBasicPhysicalDefensePercent                               //物理防御百分比
	AttributeBasicMagicalDefensePercent                                //魔法防御百分比
	AttributeBasicPhysicalCritRatePercent                              //物理暴击百分比
	AttributeBasicMagicalCritRatePercent                               //魔法暴击百分比
	AttributeBasicPhysicalCritResistancePercent                        //物理暴击抗性百分比
	AttributeBasicMagicalCritResistancePercent                         //魔法暴击抗性百分比
	AttributeBasicPhysicalHitPercent                                   //物理命中百分比
	AttributeBasicMagicalHitPercent                                    //魔法命中百分比
	AttributeBasicPhysicalDodgePercent                                 //物理闪避百分比
	AttributeBasicMagicalDodgePercent                                  //魔法闪避百分比
	AttributeBasicPhysicalPenetrationPercent                           //物理穿透百分比
	AttributeBasicMagicalPenetrationPercent                            //魔法穿透百分比
	AttributeBasicPhysicalBlockPercent                                 //物理格挡百分比
	AttributeBasicMagicalBlockPercent                                  //魔法格挡百分比
	AttributeBasicSwordsmanBaseHp                                      //剑士基础生命值
	AttributeBasicSwordsmanBasePhysicalAttack                          //剑士基础物理攻击
	AttributeBasicSwordsmanBasePhysicalDefense                         //剑士基础物理防御
	AttributeBasicSwordsmanBaseMagicalDefense                          //剑士基础魔法防御
	AttributeBasicGunnerBaseHp                                         //枪手基础生命值
	AttributeBasicGunnerBasePhysicalAttack                             //枪手基础物理攻击
	AttributeBasicGunnerBasePhysicalDefense                            //枪手基础物理防御
	AttributeBasicGunnerBaseMagicalDefense                             //枪手基础魔法防御
	AttributeBasicMageBaseHp                                           //法师基础生命值
	AttributeBasicMageBaseMagicalAttack                                //法师基础魔法攻击
	AttributeBasicMageBasePhysicalDefense                              //法师基础物理防御
	AttributeBasicMageBaseMagicalDefense                               //法师基础魔法防御
	AttributeBasicBrawlerBaseHp                                        //格斗家基础生命值
	AttributeBasicBrawlerBasePhysicalAttack                            //格斗家基础物理攻击
	AttributeBasicBrawlerBasePhysicalDefense                           //格斗家基础物理防御
	AttributeBasicBrawlerBaseMagicalDefense                            //格斗家基础魔法防御
	AttributeBasicEnvoyBaseHp                                          //使者基础生命值
	AttributeBasicEnvoyBaseMagicalAttack                               //使者基础魔法攻击
	AttributeBasicEnvoyBasePhysicalDefense                             //使者基础物理防御
	AttributeBasicEnvoyBaseMagicalDefense                              //使者基础魔法防御
	AttributeBasicSwordsmanBaseHpPercent                               //剑士基础生命值百分比
	AttributeBasicSwordsmanBasePhysicalAttackPercent                   //剑士基础物理攻击百分比
	AttributeBasicSwordsmanBasePhysicalDefensePercent                  //剑士基础物理防御百分比
	AttributeBasicSwordsmanBaseMagicalDefensePercent                   //剑士基础魔法防御百分比
	AttributeBasicGunnerBaseHpPercent                                  //枪手基础生命值百分比
	AttributeBasicGunnerBasePhysicalAttackPercent                      //枪手基础物理攻击百分比
	AttributeBasicGunnerBasePhysicalDefensePercent                     //枪手基础物理防御百分比
	AttributeBasicGunnerBaseMagicalDefensePercent                      //枪手基础魔法防御百分比
	AttributeBasicMageBaseHpPercent                                    //法师基础生命值百分比
	AttributeBasicMageBaseMagicalAttackPercent                         //法师基础魔法攻击百分比
	AttributeBasicMageBasePhysicalDefensePercent                       //法师基础物理防御百分比
	AttributeBasicMageBaseMagicalDefensePercent                        //法师基础魔法防御百分比
	AttributeBasicBrawlerBaseHpPercent                                 //格斗家基础生命值百分比
	AttributeBasicBrawlerBasePhysicalAttackPercent                     //格斗家基础物理攻击百分比
	AttributeBasicBrawlerBasePhysicalDefensePercent                    //格斗家基础物理防御百分比
	AttributeBasicBrawlerBaseMagicalDefensePercent                     //格斗家基础魔法防御百分比
	AttributeBasicEnvoyBaseHpPercent                                   //使者基础生命值百分比
	AttributeBasicEnvoyBaseMagicalAttackPercent                        //使者基础魔法攻击百分比
	AttributeBasicEnvoyBasePhysicalDefensePercent                      //使者基础物理防御百分比
	AttributeBasicEnvoyBaseMagicalDefensePercent                       //使者基础魔法防御百分比
)

// 一级属性
var FirstClassAttrIdMap = []int32{
	AttributeBasicHp,
	AttributeBasicPhysicalAttack,
	AttributeBasicMagicalAttack,
	AttributeBasicPhysicalDefense,
	AttributeBasicMagicalDefense,
	AttributeBasicStrength,
	AttributeBasicIntelligence,
	AttributeBasicToughness,
	AttributeBasicToughnessEnhancement,
	AttributeBasicToughnessResistance,
	AttributeBasicAttackSpeed,
	AttributeBasicMoveSpeed,
	AttributeBasicPhysicalCritRate,
	AttributeBasicMagicalCritRate,
	AttributeBasicPhysicalCritDamage,
	AttributeBasicMagicalCritDamage,
	AttributeBasicPhysicalCritResistance,
	AttributeBasicMagicalCritResistance,
	AttributeBasicPhysicalCritDamageResistance,
	AttributeBasicMagicalCritDamageResistance,
	AttributeBasicPhysicalHit,
	AttributeBasicMagicalHit,
	AttributeBasicPhysicalDodge,
	AttributeBasicMagicalDodge,
	AttributeBasicPhysicalPenetration,
	AttributeBasicMagicalPenetration,
	AttributeBasicPhysicalBlock,
	AttributeBasicMagicalBlock,
	AttributeBasicPhysicalDamageIncrease,
	AttributeBasicMagicalDamageIncrease,
	AttributeBasicPhysicalDamageReduction,
	AttributeBasicMagicalDamageReduction,
	AttributeBasicFinalDamageIncrease,
	AttributeBasicFinalDamageReduction,
	AttributeBasicPvpDamageIncrease,
	AttributeBasicPvpDamageReduction,
	AttributeBasicSkillAttackPower,
}

// 二级属性
var SecondClassAttrIdMap = []int32{
	AttributeBasicBaseHp,
	AttributeBasicBaseHpPercent,
	AttributeBasicBasePhysicalAttack,
	AttributeBasicBasePhysicalAttackPercent,
	AttributeBasicBaseMagicalAttack,
	AttributeBasicBaseMagicalAttackPercent,
	AttributeBasicBasePhysicalDefense,
	AttributeBasicBasePhysicalDefensePercent,
	AttributeBasicBaseMagicalDefense,
	AttributeBasicBaseMagicalDefensePercent,
	// 剑士职业基础属性
	AttributeBasicSwordsmanBaseHp,
	AttributeBasicSwordsmanBasePhysicalAttack,
	AttributeBasicSwordsmanBasePhysicalDefense,
	AttributeBasicSwordsmanBaseMagicalDefense,
	// 枪手职业基础属性
	AttributeBasicGunnerBaseHp,
	AttributeBasicGunnerBasePhysicalAttack,
	AttributeBasicGunnerBasePhysicalDefense,
	AttributeBasicGunnerBaseMagicalDefense,
	// 法师职业基础属性
	AttributeBasicMageBaseHp,
	AttributeBasicMageBaseMagicalAttack,
	AttributeBasicMageBasePhysicalDefense,
	AttributeBasicMageBaseMagicalDefense,
	// 格斗家职业基础属性
	AttributeBasicBrawlerBaseHp,
	AttributeBasicBrawlerBasePhysicalAttack,
	AttributeBasicBrawlerBasePhysicalDefense,
	AttributeBasicBrawlerBaseMagicalDefense,
	// 使者职业基础属性
	AttributeBasicEnvoyBaseHp,
	AttributeBasicEnvoyBaseMagicalAttack,
	AttributeBasicEnvoyBasePhysicalDefense,
	AttributeBasicEnvoyBaseMagicalDefense,
	// 剑士职业基础百分比属性
	AttributeBasicSwordsmanBaseHpPercent,
	AttributeBasicSwordsmanBasePhysicalAttackPercent,
	AttributeBasicSwordsmanBasePhysicalDefensePercent,
	AttributeBasicSwordsmanBaseMagicalDefensePercent,
	// 枪手职业基础百分比属性
	AttributeBasicGunnerBaseHpPercent,
	AttributeBasicGunnerBasePhysicalAttackPercent,
	AttributeBasicGunnerBasePhysicalDefensePercent,
	AttributeBasicGunnerBaseMagicalDefensePercent,
	// 法师职业基础百分比属性
	AttributeBasicMageBaseHpPercent,
	AttributeBasicMageBaseMagicalAttackPercent,
	AttributeBasicMageBasePhysicalDefensePercent,
	AttributeBasicMageBaseMagicalDefensePercent,
	// 格斗家职业基础百分比属性
	AttributeBasicBrawlerBaseHpPercent,
	AttributeBasicBrawlerBasePhysicalAttackPercent,
	AttributeBasicBrawlerBasePhysicalDefensePercent,
	AttributeBasicBrawlerBaseMagicalDefensePercent,
	// 使者职业基础百分比属性
	AttributeBasicEnvoyBaseHpPercent,
	AttributeBasicEnvoyBaseMagicalAttackPercent,
	AttributeBasicEnvoyBasePhysicalDefensePercent,
	AttributeBasicEnvoyBaseMagicalDefensePercent,
}

// 三级属性
var ThirdClassAttrIdMap = []int32{
	AttributeBasicHpPercent,
	AttributeBasicPhysicalAttackPercent,
	AttributeBasicMagicalAttackPercent,
	AttributeBasicPhysicalDefensePercent,
	AttributeBasicMagicalDefensePercent,
	AttributeBasicPhysicalCritRatePercent,
	AttributeBasicMagicalCritRatePercent,
	AttributeBasicPhysicalCritResistancePercent,
	AttributeBasicMagicalCritResistancePercent,
	AttributeBasicPhysicalHitPercent,
	AttributeBasicMagicalHitPercent,
	AttributeBasicPhysicalDodgePercent,
	AttributeBasicMagicalDodgePercent,
	AttributeBasicPhysicalPenetrationPercent,
	AttributeBasicMagicalPenetrationPercent,
	AttributeBasicPhysicalBlockPercent,
	AttributeBasicMagicalBlockPercent,
}
